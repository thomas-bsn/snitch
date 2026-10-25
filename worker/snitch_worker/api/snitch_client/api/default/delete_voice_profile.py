from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_profile_deletion import VoiceProfileDeletion
from typing import cast



def _get_kwargs(
    id: int,
    *,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/persons/{id}/voice-profile".format(id=quote(str(id), safe=""),),
    }


    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceProfileDeletion:
    if response.status_code == 200:
        response_200 = VoiceProfileDeletion.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceProfileDeletion]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceProfileDeletion]:
    """  Compte lié à la Personne, ou administrateur (sinon 403). Efface les Échantillons vocaux validés, les
    empreintes des voix attribuées à la Personne et toutes les attributions à son nom, qui redeviennent
    inconnues ; les clips, la Personne et les autres Personnes ne sont pas touchés. Un marqueur de
    retrait sans donnée vocale empêche le rapprochement de reconstituer le profil : validations et
    corrections vers cette Personne sont refusées (409 voice_profile_deleted) et une piste micro qui la
    désigne n'est plus attribuée. La suppression est tracée (compteurs, Personne, auteur) sans aucune
    donnée vocale.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceProfileDeletion]
     """


    kwargs = _get_kwargs(
        id=id,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceProfileDeletion | None:
    """  Compte lié à la Personne, ou administrateur (sinon 403). Efface les Échantillons vocaux validés, les
    empreintes des voix attribuées à la Personne et toutes les attributions à son nom, qui redeviennent
    inconnues ; les clips, la Personne et les autres Personnes ne sont pas touchés. Un marqueur de
    retrait sans donnée vocale empêche le rapprochement de reconstituer le profil : validations et
    corrections vers cette Personne sont refusées (409 voice_profile_deleted) et une piste micro qui la
    désigne n'est plus attribuée. La suppression est tracée (compteurs, Personne, auteur) sans aucune
    donnée vocale.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceProfileDeletion
     """


    return sync_detailed(
        id=id,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceProfileDeletion]:
    """  Compte lié à la Personne, ou administrateur (sinon 403). Efface les Échantillons vocaux validés, les
    empreintes des voix attribuées à la Personne et toutes les attributions à son nom, qui redeviennent
    inconnues ; les clips, la Personne et les autres Personnes ne sont pas touchés. Un marqueur de
    retrait sans donnée vocale empêche le rapprochement de reconstituer le profil : validations et
    corrections vers cette Personne sont refusées (409 voice_profile_deleted) et une piste micro qui la
    désigne n'est plus attribuée. La suppression est tracée (compteurs, Personne, auteur) sans aucune
    donnée vocale.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceProfileDeletion]
     """


    kwargs = _get_kwargs(
        id=id,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: int,
    *,
    client: AuthenticatedClient,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceProfileDeletion | None:
    """  Compte lié à la Personne, ou administrateur (sinon 403). Efface les Échantillons vocaux validés, les
    empreintes des voix attribuées à la Personne et toutes les attributions à son nom, qui redeviennent
    inconnues ; les clips, la Personne et les autres Personnes ne sont pas touchés. Un marqueur de
    retrait sans donnée vocale empêche le rapprochement de reconstituer le profil : validations et
    corrections vers cette Personne sont refusées (409 voice_profile_deleted) et une piste micro qui la
    désigne n'est plus attribuée. La suppression est tracée (compteurs, Personne, auteur) sans aucune
    donnée vocale.

    Args:
        id (int):
        origin (str):
        x_csrf_token (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceProfileDeletion
     """


    return (await asyncio_detailed(
        id=id,
client=client,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
