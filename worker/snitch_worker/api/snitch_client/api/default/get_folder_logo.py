from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...types import File, FileTypes
from ...types import UNSET, Unset
from io import BytesIO
from typing import cast



def _get_kwargs(
    *,
    name: str,
    path: str | Unset = UNSET,
    version: int | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["name"] = name

    params["path"] = path

    params["version"] = version


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/library/folder-logo",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | File:
    if response.status_code == 200:
        response_200 = File(
             payload = BytesIO(response.content)
        )



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | File]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    name: str,
    path: str | Unset = UNSET,
    version: int | Unset = UNSET,

) -> Response[Error | File]:
    """  Logo d'un dossier : l'image personnalisée choisie par un administrateur quand elle existe, sinon la
    jaquette du jeu dont le dossier porte le nom, cherchée une fois dans la boutique Steam puis
    conservée par Snitch (la navigation n'appelle jamais l'extérieur). Seul un nom égal au titre d'un
    jeu est retenu ; un nom sans jeu, un nom trop court ou la fonction désactivée (FOLDER_LOGOS=off)
    répondent 404 et le dossier garde son icône. Session requise.

    Args:
        name (str):
        path (str | Unset):
        version (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | File]
     """


    kwargs = _get_kwargs(
        name=name,
path=path,
version=version,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    name: str,
    path: str | Unset = UNSET,
    version: int | Unset = UNSET,

) -> Error | File | None:
    """  Logo d'un dossier : l'image personnalisée choisie par un administrateur quand elle existe, sinon la
    jaquette du jeu dont le dossier porte le nom, cherchée une fois dans la boutique Steam puis
    conservée par Snitch (la navigation n'appelle jamais l'extérieur). Seul un nom égal au titre d'un
    jeu est retenu ; un nom sans jeu, un nom trop court ou la fonction désactivée (FOLDER_LOGOS=off)
    répondent 404 et le dossier garde son icône. Session requise.

    Args:
        name (str):
        path (str | Unset):
        version (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | File
     """


    return sync_detailed(
        client=client,
name=name,
path=path,
version=version,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    name: str,
    path: str | Unset = UNSET,
    version: int | Unset = UNSET,

) -> Response[Error | File]:
    """  Logo d'un dossier : l'image personnalisée choisie par un administrateur quand elle existe, sinon la
    jaquette du jeu dont le dossier porte le nom, cherchée une fois dans la boutique Steam puis
    conservée par Snitch (la navigation n'appelle jamais l'extérieur). Seul un nom égal au titre d'un
    jeu est retenu ; un nom sans jeu, un nom trop court ou la fonction désactivée (FOLDER_LOGOS=off)
    répondent 404 et le dossier garde son icône. Session requise.

    Args:
        name (str):
        path (str | Unset):
        version (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | File]
     """


    kwargs = _get_kwargs(
        name=name,
path=path,
version=version,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    name: str,
    path: str | Unset = UNSET,
    version: int | Unset = UNSET,

) -> Error | File | None:
    """  Logo d'un dossier : l'image personnalisée choisie par un administrateur quand elle existe, sinon la
    jaquette du jeu dont le dossier porte le nom, cherchée une fois dans la boutique Steam puis
    conservée par Snitch (la navigation n'appelle jamais l'extérieur). Seul un nom égal au titre d'un
    jeu est retenu ; un nom sans jeu, un nom trop court ou la fonction désactivée (FOLDER_LOGOS=off)
    répondent 404 et le dossier garde son icône. Session requise.

    Args:
        name (str):
        path (str | Unset):
        version (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | File
     """


    return (await asyncio_detailed(
        client=client,
name=name,
path=path,
version=version,

    )).parsed
