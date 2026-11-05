from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_model_inspection import VoiceModelInspection
from ...models.voice_model_inspection_request import VoiceModelInspectionRequest
from typing import cast



def _get_kwargs(
    *,
    body: VoiceModelInspectionRequest,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/voice-models/inspection",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceModelInspection:
    if response.status_code == 200:
        response_200 = VoiceModelInspection.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceModelInspection]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: VoiceModelInspectionRequest,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceModelInspection]:
    """  Super-administrateur uniquement. Lit les métadonnées publiques d'un dépôt Hugging Face collé
    (owner/nom ou adresse huggingface.co ; aucun autre hôte) sans rien télécharger ni exécuter, avec
    délai et taille de réponse bornés. Propose un type seulement si les métadonnées le déclarent sans
    ambiguïté, déduit le moteur depuis la matrice d'adaptateurs approuvés et épingle le commit lu.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceModelInspectionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModelInspection]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    body: VoiceModelInspectionRequest,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceModelInspection | None:
    """  Super-administrateur uniquement. Lit les métadonnées publiques d'un dépôt Hugging Face collé
    (owner/nom ou adresse huggingface.co ; aucun autre hôte) sans rien télécharger ni exécuter, avec
    délai et taille de réponse bornés. Propose un type seulement si les métadonnées le déclarent sans
    ambiguïté, déduit le moteur depuis la matrice d'adaptateurs approuvés et épingle le commit lu.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceModelInspectionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModelInspection
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: VoiceModelInspectionRequest,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceModelInspection]:
    """  Super-administrateur uniquement. Lit les métadonnées publiques d'un dépôt Hugging Face collé
    (owner/nom ou adresse huggingface.co ; aucun autre hôte) sans rien télécharger ni exécuter, avec
    délai et taille de réponse bornés. Propose un type seulement si les métadonnées le déclarent sans
    ambiguïté, déduit le moteur depuis la matrice d'adaptateurs approuvés et épingle le commit lu.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceModelInspectionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModelInspection]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    body: VoiceModelInspectionRequest,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceModelInspection | None:
    """  Super-administrateur uniquement. Lit les métadonnées publiques d'un dépôt Hugging Face collé
    (owner/nom ou adresse huggingface.co ; aucun autre hôte) sans rien télécharger ni exécuter, avec
    délai et taille de réponse bornés. Propose un type seulement si les métadonnées le déclarent sans
    ambiguïté, déduit le moteur depuis la matrice d'adaptateurs approuvés et épingle le commit lu.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceModelInspectionRequest):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModelInspection
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
