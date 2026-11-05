from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_model import VoiceModel
from ...models.voice_model_add import VoiceModelAdd
from typing import cast



def _get_kwargs(
    *,
    body: VoiceModelAdd,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/voice-models",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceModel:
    if response.status_code == 201:
        response_201 = VoiceModel.from_dict(response.json())



        return response_201

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceModel]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: VoiceModelAdd,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceModel]:
    """  Super-administrateur uniquement. Ajoute au catalogue le dépôt Hugging Face inspecté, à la révision
    (commit) montrée par l'inspection : Snitch relit ce commit exact, vérifie le type confirmé contre
    les métadonnées et déduit lui-même le moteur. Le moteur n'est pas une entrée. Un dépôt sans
    adaptateur approuvé, ou dont le moteur ne sait pas exécuter le type choisi, est enregistré non pris
    en charge avec sa raison et ne peut pas être installé. Refusé : voice_model_kind_mismatch (le dépôt
    déclare un autre type), voice_model_exists, voice_model_hf_not_found, voice_model_hf_denied,
    voice_model_hf_unavailable.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceModelAdd):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModel]
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
    body: VoiceModelAdd,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceModel | None:
    """  Super-administrateur uniquement. Ajoute au catalogue le dépôt Hugging Face inspecté, à la révision
    (commit) montrée par l'inspection : Snitch relit ce commit exact, vérifie le type confirmé contre
    les métadonnées et déduit lui-même le moteur. Le moteur n'est pas une entrée. Un dépôt sans
    adaptateur approuvé, ou dont le moteur ne sait pas exécuter le type choisi, est enregistré non pris
    en charge avec sa raison et ne peut pas être installé. Refusé : voice_model_kind_mismatch (le dépôt
    déclare un autre type), voice_model_exists, voice_model_hf_not_found, voice_model_hf_denied,
    voice_model_hf_unavailable.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceModelAdd):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModel
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
    body: VoiceModelAdd,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceModel]:
    """  Super-administrateur uniquement. Ajoute au catalogue le dépôt Hugging Face inspecté, à la révision
    (commit) montrée par l'inspection : Snitch relit ce commit exact, vérifie le type confirmé contre
    les métadonnées et déduit lui-même le moteur. Le moteur n'est pas une entrée. Un dépôt sans
    adaptateur approuvé, ou dont le moteur ne sait pas exécuter le type choisi, est enregistré non pris
    en charge avec sa raison et ne peut pas être installé. Refusé : voice_model_kind_mismatch (le dépôt
    déclare un autre type), voice_model_exists, voice_model_hf_not_found, voice_model_hf_denied,
    voice_model_hf_unavailable.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceModelAdd):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModel]
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
    body: VoiceModelAdd,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceModel | None:
    """  Super-administrateur uniquement. Ajoute au catalogue le dépôt Hugging Face inspecté, à la révision
    (commit) montrée par l'inspection : Snitch relit ce commit exact, vérifie le type confirmé contre
    les métadonnées et déduit lui-même le moteur. Le moteur n'est pas une entrée. Un dépôt sans
    adaptateur approuvé, ou dont le moteur ne sait pas exécuter le type choisi, est enregistré non pris
    en charge avec sa raison et ne peut pas être installé. Refusé : voice_model_kind_mismatch (le dépôt
    déclare un autre type), voice_model_exists, voice_model_hf_not_found, voice_model_hf_denied,
    voice_model_hf_unavailable.

    Args:
        origin (str):
        x_csrf_token (str):
        body (VoiceModelAdd):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModel
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
